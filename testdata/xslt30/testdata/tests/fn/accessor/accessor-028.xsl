<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of fn:base-uri accessor with context node set to document, element, attr, txt, comment, and PI nodes (the xml source contains xml:base attributes overwriting base-uri of child node) -->

   <t:template match="/">
      <out>
         <doc>base_uri="<t:value-of select="ends-with(base-uri(.), 'accessor018.xml')"/>"</doc>
         <t:apply-templates select="*"/>
         <t:text>|done1|</t:text>
      </out>
   </t:template>

   <t:template match="doc">
      <Elmt>base_uri="<t:value-of select="base-uri(.)"/>"</Elmt>
	     <t:apply-templates select="attribute()[name(.)='att2']|node()"/>
      <t:text>|up|</t:text>
      <t:for-each select="namespace::*[name(.)='xml']">
         <t:value-of select="base-uri(.)"/>
      </t:for-each>
   </t:template>

   <t:template match="element()">
      <Elmt>base_uri="<t:value-of select="base-uri(.)"/>"</Elmt>
      <t:apply-templates select="attribute()[name(.)='att2']|node()"/>
      <t:text>|up|</t:text>
   </t:template>

   <t:template match="processing-instruction()">
      <PI>base_uri="<t:value-of select="base-uri(.)"/>"</PI>
   </t:template>

   <t:template match="attribute()[name(.)='att2']">
  		  <Attr>base_uri="<t:value-of select="base-uri(.)"/>"</Attr>
   </t:template>

   <t:template match="text()">
      <Txt>base_uri="<t:value-of select="base-uri(.)"/>"</Txt>  
   </t:template>

   <t:template match="comment()">
      <Cmt>base_uri="<t:value-of select="base-uri(.)"/>"</Cmt>
   </t:template>
</t:transform>
