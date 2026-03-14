<?xml version="1.0"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test that whitespace text nodes as children of xsl:apply-imports which 
  has @xml:space=preserve do not interfere with processing of the instruction. -->


 <t:import href="import1.xsl"/>

 <t:template match="doc">
  <out>
   <b>primary stylesheet</b>
   <t:apply-imports xml:space="preserve">
		
			&#x20; &#x9; &#xD;  &#xA;
			   
		</t:apply-imports>
  </out>
 </t:template>

</t:transform>
