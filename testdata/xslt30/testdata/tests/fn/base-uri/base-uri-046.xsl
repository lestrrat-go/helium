<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Test with base-uri($arg) where $arg is a deep copy of an 
  				element node which has an @xml:base=relative-URI in the source file. 
  				The copy of the element node has no parent.-->

   <t:variable name="elemcopy" as="element()">
	     <t:copy-of select="/doc/str1"/>
   </t:variable>

   <t:template match="/doc">
	     <out>
         <t:value-of select="base-uri($elemcopy)"/>
      </out>
   </t:template>
</t:transform>
