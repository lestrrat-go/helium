<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="xs"
   version="2.0">
   
   <xsl:template name="t050">
      <!--<description>Test of fn:path applied to document node</description>-->
      <xsl:value-of select="path(/)"/>
   </xsl:template>
   
   <xsl:template name="t051">
      <!--<description>Test of fn:path applied to outermost element node</description>-->
      <xsl:value-of select="path(/*)"/>
   </xsl:template>
   
   <xsl:template name="t052">
      <!--<description>Test of fn:path applied to inner element node</description>-->
      <xsl:value-of select="path(/*/*:inner)"/>
   </xsl:template>
   
   <xsl:template name="t053">
      <!--<description>Test of fn:path applied to comment node</description>-->
      <xsl:value-of select="path((//comment())[1])"/>
   </xsl:template>
   
   <xsl:template name="t054">
      <!--<description>Test of fn:path applied to PI node</description>-->
      <xsl:value-of select="path((//processing-instruction())[1])"/>
   </xsl:template>
   
   <xsl:template name="t055">
      <!--<description>Test of fn:path applied to text node</description>-->
      <xsl:value-of select="path((//text())[1])"/>
   </xsl:template>
   
   <xsl:template name="t056">
      <!--<description>Test of fn:path applied to attribute node</description>-->
      <xsl:value-of select="path((//@att)[1])"/>
   </xsl:template>
   
   <xsl:template name="t057">
      <!--<description>Test of fn:path applied to namespace node</description>-->
      <xsl:value-of select="path((//namespace::mynamespace)[1])"/>
   </xsl:template>
   
   <xsl:variable name="orphan" as="element()">
      <xsl:copy-of select="/*"/>
   </xsl:variable>
   
   <xsl:template name="t058">
      <!--<description>Test of fn:path applied to outermost element node in non-document tree</description>-->
      <xsl:value-of select="path($orphan)"/>
   </xsl:template>
   
   <xsl:template name="t059">
      <!--<description>Test of fn:path applied to inner element node in non-document tree</description>-->
      <xsl:value-of select="path($orphan/*:inner)"/>
   </xsl:template>
   
   <xsl:template name="t060">
      <!--<description>Test of fn:path applied to comment node in non-document tree</description>-->
      <xsl:value-of select="path(($orphan//comment())[1])"/>
   </xsl:template>
   
   <xsl:template name="t061">
      <!--<description>Test of fn:path applied to PI node in non-document tree</description>-->
      <xsl:value-of select="path(($orphan//processing-instruction())[1])"/>
   </xsl:template>
   
   <xsl:template name="t062">
      <!--<description>Test of fn:path applied to text node in non-document tree</description>-->
      <xsl:value-of select="path(($orphan//text())[1])"/>
   </xsl:template>
   
   <xsl:template name="t063">
      <!--<description>Test of fn:path applied to attribute node in non-document tree</description>-->
      <xsl:value-of select="path(($orphan//@att)[1])"/>
   </xsl:template>
   
   <xsl:template name="t064">
      <!--<description>Test of fn:path applied to namespace node in non-document tree</description>-->
      <xsl:value-of select="path(($orphan//namespace::mynamespace)[1])"/>
   </xsl:template>
</xsl:stylesheet>