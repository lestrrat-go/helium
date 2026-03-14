<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Test static-base-uri() in different parts of the stylesheet where the base URI varies.-->

   <xsl:template name="xsl:initial-template">
      <out>
         <initial base="{static-base-uri()}"/>
         <xsl:call-template name="a"/>
      </out>
   </xsl:template>
   
   <xsl:template name="a" xml:base="a">
         <a base="{static-base-uri()}"/>
         <xsl:call-template name="b"/>     
   </xsl:template>
   
   <xsl:template name="b" xml:base="b">
      <b base="{static-base-uri()}"/>
      <xsl:call-template name="c"/>     
   </xsl:template>
   
   <xsl:template name="c" xml:base="c">
      <c base="{static-base-uri()}"/>     
   </xsl:template>
</xsl:transform>
