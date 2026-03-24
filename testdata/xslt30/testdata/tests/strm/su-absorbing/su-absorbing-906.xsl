<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">


  <xsl:strip-space elements="*"/>



  <!-- Non-streamable absorbing function - consuming reference in a loop -->
  <!-- One could imagine streaming this, but the spec doesn't allow it -->
  
  <xsl:function name="f:enumerate" as="xs:string*" streamability="absorbing">
    <xsl:param name="element" as="node()*"/>
    <xsl:sequence select="for $i in 1 to 3 return name($element[$i])"/>
  </xsl:function>
  
  <xsl:template name="main" >
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="f:enumerate(/BOOKLIST/BOOKS/ITEM/*)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
