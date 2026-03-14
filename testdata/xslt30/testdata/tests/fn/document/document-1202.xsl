<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to verify that if doc-available($uri) returns true, then calling doc($uri) 
  				within the same execution scope returns a document node-->

   <t:variable name="var" select="/doc/str1"/>

   <t:template match="/doc">
      <out>
         <t:if test="doc-available($var)">
		          <t:copy-of select="doc($var)/outer"/>
	        </t:if>
      </out>
   </t:template>
</t:transform>
