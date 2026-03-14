<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Show that root(node) can be applied to the root node, which is a document node -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="snap" select="generate-id(/)"/>

   <t:template match="/">
	     <t:apply-templates select="doc"/>
	  </t:template>

   <t:template match="doc">
	     <out>
         <t:choose>
	           <t:when test="generate-id(root(/)) = $snap">
	              <t:text>Success</t:text>
	           </t:when>
	           <t:otherwise>
	              <t:text>FAIL! Original snapshot: </t:text>
	              <t:value-of select="$snap"/>
	              <t:text>, root(/) returned: </t:text>
	              <t:value-of select="generate-id(root(/))"/>
	           </t:otherwise>
	        </t:choose>
      </out>
	  </t:template>
</t:transform>
