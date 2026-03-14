<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:choose which 
  				has @xml:space=preserve do not interfere with processing of the instruction. -->

   <t:template match="/doc">
	     <out>
         <t:choose xml:space="preserve">
			  
				 
			&#xD;  
			

						
			<t:when test="false()"><b>false</b></t:when>
			
			  	 &#xD;  

			
			<t:when test="true()"><a>true</a></t:when>
			
			<t:otherwise><c>false</c></t:otherwise>
			
			  	 &#xD;  

		</t:choose>
      </out>
   </t:template>
</t:transform>
