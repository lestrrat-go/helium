<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test implicit static-base-uri() in different parts of the stylesheet where the base URI varies.-->

   <xsl:template name="xsl:initial-template">
      <out>
         <initial base="{static-base-uri()}"/>
         <xsl:call-template name="a"/>
      </out>
   </xsl:template>
   
   <xsl:template name="a" xml:base="baseuri023.xml">
         <a base="{base-uri(document(''))}"/>
         <xsl:call-template name="b"/>     
   </xsl:template>
   
   <xsl:template name="b" xml:base="baseuri025.xml">
      <b base="{base-uri(document(''))}"/>
      <xsl:call-template name="c"/>     
   </xsl:template>
   
   <xsl:template name="c" xml:base="baseuri027.xml">
      <c base="{base-uri(document(''))}"/>     
   </xsl:template>
</xsl:transform>
