<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test that whitespace text nodes as children of xsl:analyze-string which has @xml:space=preserve do not 
				interfere with processing of the instruction. -->

   <t:template match="doc">
      <out>
         <E1>
            <t:analyze-string select="'abracadabra'" regex="bra" xml:space="preserve">
    		
    	  	 &#xD;  

    	         
    	<t:matching-substring><t:value-of select="'*'"/></t:matching-substring>
    	
    	  	 &#xD;  

    	<t:non-matching-substring><t:value-of select="."/></t:non-matching-substring>
    	</t:analyze-string>
         </E1>
      </out>
   </t:template>
</t:transform>
