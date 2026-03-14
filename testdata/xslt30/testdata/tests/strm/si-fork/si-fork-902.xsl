<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- Non-streamable fork: one prong is not streamable -->
  
  <xsl:template name="f-902" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:fork>
            <xsl:sequence select="string(AUTHOR)"/>
            <xsl:sequence select="TITLE||PRICE"/>
          </xsl:fork>
        </xsl:for-each>
      </out>  
    </xsl:source-document>
  </xsl:template>
  
  
</xsl:transform>  