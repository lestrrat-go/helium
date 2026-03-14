<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of 
  				xsl:next-match which has @xml:space=preserve do 
  				not interfere with processing of the instruction. -->

   <t:template match="doc">
	     <out>
         <b>primary stylesheet</b>
         <t:next-match xml:space="preserve">
		
			  	 &#xD;  

			   
		</t:next-match>
      </out>
   </t:template>

   <t:template match="*">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform"> next match </a>
   </t:template>
</t:transform>
