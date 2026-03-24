<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:character-map 
  				which has @xml:space=preserve do not interfere with processing of the instruction. -->

   <t:character-map name="temp" xml:space="preserve">
		  
			 &#xD;  
		

  <t:output-character character="a" string="b"/>
  	 &#xD;   
</t:character-map>

   <t:template match="doc">
	     <t:result-document use-character-maps="temp">
		       <out>bababa</out>
	     </t:result-document>
   </t:template>
</t:transform>
