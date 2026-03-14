<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of fn:string accessor with doc node, element, attr, txt, comment, and PI nodes. -->

   <t:template match="/">
      <out>
         <t:apply-templates select="*"/>
         <t:text>|done|</t:text>
      </out>
   </t:template>

   <t:template match="doc">
      <t:text>
</t:text>
      <Elmt1>name="<t:value-of select="string()"/>"</Elmt1>
	     <t:text>|</t:text>
      <Elmt2>name="<t:value-of select="string(.)"/>"</Elmt2>
      <t:apply-templates select="@*|node()"/>
      <t:text>|up|</t:text>
      <t:for-each select="namespace::*[name(.)='sub']">
         <t:value-of select="string(.)"/>
      </t:for-each>
   </t:template>

   <t:template match="element()">
      <t:text>
</t:text>
      <Elmt1>name="<t:value-of select="string()"/>"</Elmt1>
      <t:text>|</t:text>
      <Elmt2>name="<t:value-of select="string(.)"/>"</Elmt2>
      <t:apply-templates select="@*|node()"/>
      <t:text>|up|</t:text>
   </t:template>

   <t:template match="processing-instruction()">
      <PI1>name="<t:value-of select="string()"/>"</PI1>
      <PI2>name="<t:value-of select="string(.)"/>"</PI2>
   </t:template>

   <t:template match="attribute()">
      <Attr1>name="<t:value-of select="string()"/>"</Attr1>
      <Attr2>name="<t:value-of select="string(.)"/>"</Attr2>
   </t:template>

   <t:template match="text()">
      <Txt1>name="<t:value-of select="string()"/>"</Txt1>
      <Txt2>name="<t:value-of select="string(.)"/>"</Txt2>
   </t:template>

   <t:template match="comment()">
      <Cmt1>name="<t:value-of select="string()"/>"</Cmt1>
      <Cmt2>name="<t:value-of select="string(.)"/>"</Cmt2>
   </t:template>
</t:transform>
