<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f/">

  <!-- PURPOSE: Test handling of }}} in an AVT. -->
  <xsl:template match="/">
    <xsl:variable name="subtarget" select="'fofofo'"/>
    <xsl:variable name="subproj" select="'rarara'"/>
    <xsl:variable name="script" select="'mymymy'"/>
    
    <ant target="{$subtarget}" inheritall="false" 
     antfile="${{project.{$subproj}}}/{$script}"/>
  </xsl:template>
  

 
</xsl:stylesheet>
