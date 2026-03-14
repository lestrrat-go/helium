<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- non-streamable use of current() in predicate -->
    
    <xsl:template name="main" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:for-each select="head(/BOOKLIST/BOOKS/ITEM)">
            <xsl:copy-of select="*[string(.) = string(current())]"/>
          </xsl:for-each>  
        </out>
      </xsl:source-document>
    </xsl:template>
    

    
</xsl:stylesheet>