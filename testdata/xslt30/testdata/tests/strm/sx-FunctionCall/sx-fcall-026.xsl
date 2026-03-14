<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local.functions/"
    exclude-result-prefixes="map xs f">
    
    <!-- Test of xsl:stream with max of a computed value
         Can't call a function supplying a streamed node. 
         Although the function call isn't streamable in the W3C spec, Saxon inlines it.
         So the remaining problem is that there are two downward selections. -->
    
    <xsl:template name="main">
      <xsl:source-document streamable="yes" href="books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM[f:largestSide(.) gt 20]/TITLE" separator=","/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <xsl:function name="f:largestSide" as="xs:decimal">
      <xsl:param name="item" as="element(ITEM)"/>
      <xsl:sequence select="max($item!DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
    </xsl:function>   
    
</xsl:stylesheet>