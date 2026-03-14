<?xml version="1.0"?>

<!-- on-no-match="text-only-copy" --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  <xsl:template match="/"><out><xsl:apply-templates/></out></xsl:template>
 
  <xsl:mode name="s" on-no-match="text-only-copy"/>
   
</xsl:stylesheet>