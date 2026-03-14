<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Test with base-uri($arg) where $arg is a copied element node attached 
  				as a child to a newly created document node with no xml:base attribute.-->

   <t:template match="/doc">
	     <out>
         <t:apply-templates select="str1/*"/>
      </out>
   </t:template>

   <t:template match="substring1">
	
	     <t:variable name="elemcopy">
		       <t:copy>new content</t:copy>
	     </t:variable>
		
	     <t:value-of select="base-uri($elemcopy/substring1)"/>		
   </t:template>
</t:transform>
