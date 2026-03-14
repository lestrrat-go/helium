<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Test with base-uri($arg) and two nested xsl:copy instructions. 
  				Base-uri of second copied node is the base-uri of its parent -the first copied node.-->

   <t:template match="/doc">
	     <out>
         <t:apply-templates select="str1/*"/>
      </out>
   </t:template>

   <t:template match="substring1">
	     <t:variable name="elemcopy" xml:base="/xxx/">
		       <e1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
            <t:copy>
				           <t:apply-templates select="/doc/str1"/>
			         </t:copy>
         </e1>
	     </t:variable>
	
	     <t:value-of select="$elemcopy//*/name()"/>
      <t:text> *** </t:text>
	     <t:value-of select="base-uri($elemcopy/e1/substring1/str1)"/>	
   </t:template>

   <t:template match="str1">
		    <t:copy>original parent</t:copy>
   </t:template>
</t:transform>
