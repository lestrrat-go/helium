<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Tests static variables and non-static variables with the same name (spec unclear) -->
   
   <xsl:import href="static-020a.xsl"/>
   
   <xsl:variable name="p" static="no" select="0"/>
  
   <xsl:variable name="q" static="yes" select="$p+10"/>
   
   <xsl:template name="xsl:initial-template">
      <result p="{$p}" q="{$q}"/>
   </xsl:template>
   
</xsl:transform>
