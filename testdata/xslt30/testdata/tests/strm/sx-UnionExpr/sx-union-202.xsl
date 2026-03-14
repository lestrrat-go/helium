<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
 
 
 
  <!-- The union of two striding expressions is crawling -->
  
  <xsl:template name="xsl:initial-template" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:apply-templates select="(/BOOKLIST/ITEM | /BOOKLIST/MAGAZINE)/PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:mode streamable="yes" on-no-match="shallow-copy"/>

            
    
</xsl:stylesheet>