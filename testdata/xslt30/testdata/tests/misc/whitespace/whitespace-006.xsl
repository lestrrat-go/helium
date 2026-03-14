<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:attribute-set which 
  				has @xml:space=preserve do not interfere with processing of the instruction. -->

   <t:attribute-set name="set1" xml:space="preserve">
			
	  	 &#xD;  


  <t:attribute name="color">red</t:attribute>
  					        	  
  			<t:attribute name="number">2</t:attribute>
</t:attribute-set>

   <t:template match="/">
      <out>
         <item t:use-attribute-sets="set1"/>
      </out>
   </t:template>
</t:transform>
