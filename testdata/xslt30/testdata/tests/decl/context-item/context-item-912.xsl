<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- xsl:context-item inconsistent with the match pattern,
         leading to a type error which is detectable statically -->
  
  <!-- §10.1.1 - "The processor may signal a type error statically if the required context item type 
    is incompatible with the match pattern, that is, if no item that satisfies the match 
    pattern can also satisfy the required context item type." -->
  
    <xsl:template match="*">
      <xsl:context-item use="required" as=" xs:integer "/>
      <xsl:sequence select="."/>
    </xsl:template>
    
    <xsl:template name="main">
      <xsl:variable name="tree"><tree/></xsl:variable>
      <out>
         <xsl:apply-templates select="$tree"/>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>