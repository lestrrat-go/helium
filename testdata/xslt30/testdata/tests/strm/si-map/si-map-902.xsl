<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
  
  <!-- within xsl:stream, map key and value are both consuming -->
  
  <xsl:template name="m-902" use-when="$RUN" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="head(//AUTHOR)" select="data(head(//TITLE))"/>
            </xsl:map>
          </xsl:variable>
          <authors>{count($m('Jane Austen'))}</authors>    
      </xsl:source-document>
    </out>
  </xsl:template>  
  


  
</xsl:transform>  