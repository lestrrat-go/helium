<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- xsl:context-item supplying more type information than is present in the match pattern,
         leading to a type error which is detectable statically -->
  
    <xsl:template match=".">
      <xsl:context-item use="required" as=" xs:integer "/>
      <xsl:sequence select="./child-element"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:apply-templates select="."/>
         </xsl:for-each>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>