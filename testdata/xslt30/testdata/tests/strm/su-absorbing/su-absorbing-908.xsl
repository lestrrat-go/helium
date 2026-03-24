<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">


  <xsl:strip-space elements="*"/>



  <!-- Non-streamable absorbing function - multiple consuming variable references -->
  
  <xsl:function name="f:count-descendants-or-self" as="xs:integer" streamability="absorbing">
    <xsl:param name="input" as="node()*"/>
    <xsl:sequence select="if (empty($input)) then 0 else 1 + f:count-descendants-or-self($input/*)"/>
  </xsl:function>
  
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-or-self(/BOOKLIST/BOOKS/ITEM)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

</xsl:stylesheet>
