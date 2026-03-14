<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:call-template which 
  				has @xml:space=preserve do not interfere with processing of the instruction. -->

   <t:template match="/doc">
	     <out>
         <b>template for doc</b>
         <t:call-template name="temp" xml:space="preserve">
		
			  	 &#xD;  

			   
		</t:call-template>
      </out>
   </t:template>

   <t:template name="temp">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform"> temp template </a>
   </t:template>
</t:transform>
