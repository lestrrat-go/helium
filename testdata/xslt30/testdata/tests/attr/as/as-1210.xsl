<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of several templates which contain document, element,attribute, 
				comment, text or processing-instruction nodes and have @as="node()". -->

   <t:template match="/">
      <out>
         <e1>
            <t:call-template name="a1"/>
         </e1>
         <e2>
            <t:call-template name="a2"/>
         </e2>
         <e3>
            <t:call-template name="a3"/>
         </e3>
         <e4>
            <t:call-template name="a4"/>
         </e4>
         <e5>
            <t:call-template name="a5"/>
         </e5>
         <e6>
            <t:call-template name="a6"/>
         </e6>
         <e21>
            <t:call-template name="a21"/>
         </e21>
         <e31>
            <t:call-template name="a31"/>
         </e31>
         <e41>
            <t:call-template name="a41"/>
         </e41>
         <e51>
            <t:call-template name="a51"/>
         </e51>
         <e61>
            <t:call-template name="a61"/>
         </e61>
      </out>
   </t:template>

   <t:template name="a1" as="node()">
      <t:document>
         <item>hello</item>
      </t:document>
   </t:template>

   <t:template name="a2" as="node()">
      <t:copy-of select="doc/item"/>
   </t:template>

   <t:template name="a3" as="node()">
      <t:copy-of select="doc/item/@attrib"/>
   </t:template>

   <t:template name="a4" as="node()">
      <t:copy-of select="/doc/processing-instruction()"/>
   </t:template>

   <t:template name="a5" as="node()">
      <t:copy-of select="/doc/item/text()"/>
   </t:template>

   <t:template name="a6" as="node()">
      <t:copy-of select="/doc/item/comment()"/>
   </t:template>

   <t:template name="a21" as="node()">
      <t:element name="e1">element</t:element>
   </t:template>

   <t:template name="a31" as="node()">
      <t:attribute name="e1">attribute</t:attribute>
   </t:template>

   <t:template name="a41" as="node()">
      <t:processing-instruction name="e1">PI</t:processing-instruction>
   </t:template>

   <t:template name="a51" as="node()">
      <t:text>text</t:text>
   </t:template>

   <t:template name="a61" as="node()">
      <t:comment>comment</t:comment>
   </t:template>
</t:transform>
