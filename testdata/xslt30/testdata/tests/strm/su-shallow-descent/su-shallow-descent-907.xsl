<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">


  <xsl:strip-space elements="*"/>



  <!-- Non-streamable shallow-descent function - second argument is consuming -->
  
  <xsl:function name="f:g"  streamability="shallow-descent">
      <xsl:param name="n" as="node()"/>
      <xsl:param name="o"/>
      <xsl:sequence select="$n/*[empty($o)]"/>
  </xsl:function>
  
  <xsl:template name="main" >
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
          <xsl:sequence select="f:g(., *)"/>
        </xsl:for-each>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
