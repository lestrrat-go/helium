<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.whitespace.example.com/ns/various"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test that whitespace text nodes are stripped from elements with element 
  				only content regardless of xsl:preserve-space or xsl:strip-space. -->

   <t:preserve-space elements="my:complexUserElem"/>

   <t:template match="/my:userNode">
      <t:variable name="copy">
	        <out>
	           <t:copy-of select="my:complexUserElem[1]"/>
         </out>
	     </t:variable>
	     <t:choose>
	        <t:when test="exists($copy//text()[matches(.,'\s+')])">
	           <t:copy-of select="$copy"/>
	        </t:when>
	        <t:otherwise>
	           <ok/>
	        </t:otherwise>
	     </t:choose>
   </t:template>
</t:transform>
