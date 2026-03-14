<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:apply-templates which 
  				has @xml:space=preserve do not interfere with processing of the instruction. -->

   <t:template match="/">
	     <out>
         <b>template for root</b>
         <t:apply-templates select="doc" xml:space="preserve">
		
			  	 &#xD;  

			   
		</t:apply-templates>
      </out>
   </t:template>

   <t:template match="doc">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform"> template for doc </a>
   </t:template>
</t:transform>
