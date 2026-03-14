<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <?spec xslt#patterns?>
        <!-- Purpose: apply-imports with atomic value as context item -->
      
    <xsl:import href="match-133a.xsl"/>  
    
    <xsl:template name="main">
     <out>
     	<xsl:apply-templates select="17"/>
     </out>
    </xsl:template>
    
    <xsl:template match=".[. gt 0]">
      <first><xsl:apply-imports/></first>
    </xsl:template>



</xsl:stylesheet>
