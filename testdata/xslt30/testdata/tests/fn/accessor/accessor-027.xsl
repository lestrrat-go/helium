<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:base-uri accessor with context node set to document, element, attr, txt, comment, and PI nodes (the xml source contains no xml:base attributes) -->

   <xslt:template match="/">
      <out>
         <doc>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</doc>
         <xslt:apply-templates select="*"/>
         <xslt:text>|done1|</xslt:text>
      </out>
   </xslt:template>

   <xslt:template match="doc">
      <Elmt>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</Elmt>
	     <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
      <xslt:for-each select="namespace::*">
         <xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>
      </xslt:for-each>
   </xslt:template>

   <xslt:template match="element()">
      <Elmt>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</Elmt>
      <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
   </xslt:template>

   <xslt:template match="processing-instruction()">
      <PI>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</PI>
   </xslt:template>

   <xslt:template match="@*">
      <Attr>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</Attr>
   </xslt:template>

   <xslt:template match="text()">
      <Txt>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</Txt>  
   </xslt:template>

   <xslt:template match="comment()">
      <Cmt>base_uri="<xslt:value-of select="ends-with(base-uri(.), 'accessor017.xml')"/>"</Cmt>
   </xslt:template>
</xslt:transform>
