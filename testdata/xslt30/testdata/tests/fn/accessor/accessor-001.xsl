<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of node-name() accessor with document, element, attr, txt, comment, and PI nodes -->

   <t:template match="/">
      <out>
         <doc_node>doc_name="<t:value-of select="node-name(.)"/>"
    </doc_node>
         <t:apply-templates select="*"/>
         <t:text>|done|</t:text>
      </out>
   </t:template>

   <t:template match="doc">
      <t:text>
</t:text>
      <Elmt>name="<t:value-of select="node-name(.)"/>"
  </Elmt>
      <t:apply-templates select="@*|node()"/>
      <t:text>|up|</t:text>
      <t:for-each select="namespace::*[name(.)='mynamespace']">
         <t:value-of select="node-name(.)"/>
      </t:for-each>

   </t:template>

   <t:template match="element()">
      <t:text>
</t:text>
      <Elmt>name="<t:value-of select="node-name(.)"/>"
  </Elmt>
      <t:apply-templates select="@*|node()"/>
      <t:text>|up|</t:text>
   </t:template>

   <t:template match="processing-instruction()">
      <PI>name="<t:value-of select="node-name(.)"/>"
  </PI>
   </t:template>

   <t:template match="attribute()">
      <Attr>name="<t:value-of select="node-name(.)"/>"
  </Attr>
   </t:template>

   <t:template match="text()">
  
      <Txt>name="<t:value-of select="node-name(.)"/>"
  </Txt>
   </t:template>

   <t:template match="comment()">
      <Cmt>name="<t:value-of select="node-name(.)"/>"
  </Cmt>
   </t:template>
</t:transform>
