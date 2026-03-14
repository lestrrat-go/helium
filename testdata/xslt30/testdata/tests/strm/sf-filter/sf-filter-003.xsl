<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://www.w3.org/xslt30tests/filter"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:function name="f:test" as="xs:boolean">
      <xsl:param name="arg" as="item()"/>
      <xsl:sequence select="number($arg) gt 10"/>
    </xsl:function>  
    
    <!-- Streaming filter(): grounded atomic operand -->
    
    <xsl:template name="r-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/DIMENSIONS ! filter(tokenize(., ' '), f:test#1)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
            
    
</xsl:stylesheet>