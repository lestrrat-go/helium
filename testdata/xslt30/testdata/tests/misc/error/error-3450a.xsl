<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--
      It is a static error if a variable declared with static="yes" is inconsistent with 
      another static variable of the same name that is declared earlier in stylesheet tree order and 
      that has lower import precedence.
-->
   
   <xsl:import href="error-3450a2.xsl"/>
   
   <xsl:variable name="v" static="yes" select="2"/>
   
   <xsl:template name="main">
      <out>
         <x v="{$v}"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
