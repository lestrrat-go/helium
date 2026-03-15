<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Test with base-uri($arg) where $arg is a deep copy of an  element 
  				node attached as a child to a newly created document node with no 
  				xml:base attribute.-->

   <t:variable name="elemcopy">
	     <t:copy-of select="/doc/str1/substring1"/>
   </t:variable>

   <t:template match="/doc">
	     <out>
         <t:value-of select="base-uri($elemcopy/substring1)"/>
      </out>
   </t:template>
</t:transform>
