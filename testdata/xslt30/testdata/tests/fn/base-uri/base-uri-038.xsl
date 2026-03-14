<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" xml:base="http://www.example.org/"
   version="2.0">
   <!-- Purpose: Test with base-uri($arg) where $arg is a copied element node attached as a child to a newly created 
  				document node, containing variable has an explicit xml:base attribute, and the copied node's parent 
  				had an xml:base=relative-URI attribute in the source file.-->

   <t:template match="/doc">
      <out>
         <t:apply-templates select="str1/*"/>
      </out>
   </t:template>

   <t:template match="substring1">
      <t:variable name="e" xml:base="/main/">
         <t:copy>new content</t:copy>
      </t:variable>
      <t:value-of select="base-uri($e/substring1)"/>
   </t:template>
</t:transform>
