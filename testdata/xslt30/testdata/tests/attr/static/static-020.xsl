<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Tests static attribute with multiple import precedences -->
   
   <xsl:import href="static-020a.xsl"/>
   
   <xsl:variable name="v1" _select="{$p}"/>
   
   <xsl:param name="p" static="yes" select="0"/>
   
   <xsl:variable name="v2" _select="{$p}"/>
   
   <xsl:import href="static-020b.xsl"/>
   
   <xsl:variable name="v3" _select="{$p}"/>
   
   <xsl:import href="static-020a.xsl"/>
   
   <xsl:variable name="v4" _select="{$p}"/>
   
   <xsl:template name="xsl:initial-template" expand-text="yes">
      <result v1="{$v1}" v2="{$v2}" v3="{$v3}" v4="{$v4}"/>
   </xsl:template>
   
</xsl:transform>
