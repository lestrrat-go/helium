<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" xml:base="http://www.example.org/" version="2.0">
   <!-- Purpose: Test with base-uri($arg) where $arg is a copied element node attached as a child to a newly 
				created element with @xml:base and the copied node had an @xml:base=absolute-URI in the source file.-->


   <t:template match="substring1">
      <t:variable name="e" xml:base="/xxx/">
         <e1 xml:base="/main/">
            <t:copy>new content</t:copy>
         </e1>
      </t:variable>
      <t:value-of select="base-uri($e/e1/substring1)"/>
      <t:text> , </t:text>
      <t:value-of select="base-uri($e)"/>
   </t:template>
   
   <t:template match="/doc">
      <out>
         <t:apply-templates select="str1/*"/>
      </out>
   </t:template>
   
</t:transform>
