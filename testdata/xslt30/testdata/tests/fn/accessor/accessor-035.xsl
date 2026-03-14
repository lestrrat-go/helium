<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of fn:document-uri accessor with context node set to document node, element, atttribute, comment, text, and PI as argument -->

   <t:template match="/">
      <out>
         <doc>doc_uri="<t:value-of select="ends-with(document-uri(.), 'accessor023.xml')"/>"</doc>
         <t:apply-templates select="*"/>
         <t:text>|done1|</t:text>
      </out>
   </t:template>

   <t:template match="element()">
      <Elmt>base_uri="<t:value-of select="document-uri(.) instance of empty-sequence()"/>"</Elmt>
      <t:apply-templates select="@*|node()"/>
      <t:text>|up|</t:text>
   </t:template>

   <t:template match="processing-instruction()">
      <PI>base_uri="<t:value-of select="document-uri(.) instance of empty-sequence()"/>"</PI>
   </t:template>

   <t:template match="@*">
      <Attr>base_uri="<t:value-of select="document-uri(.) instance of empty-sequence()"/>"</Attr>
   </t:template>

   <t:template match="text()">
      <Txt>base_uri="<t:value-of select="document-uri(.) instance of empty-sequence()"/>"</Txt>
   </t:template>

   <t:template match="comment()">
      <Cmt>base_uri="<t:value-of select="document-uri(.) instance of empty-sequence()"/>"</Cmt>
   </t:template>
</t:transform>
