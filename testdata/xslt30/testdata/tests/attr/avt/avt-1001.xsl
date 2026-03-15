<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in @stable of xsl:sort. AVT value has 
  				no literal element and comes from a node.-->

   <t:template match="/doc">
      <out>
         <t:for-each select="item">
  		        <t:sort lang="en" stable="{/doc/case1}"/>
  		        <t:sort select="@attr" lang="en" case-order="lower-first"/>
  		        <t:value-of select="@num"/>
  		        <t:value-of select="."/>
            <t:text>@</t:text>
  		        <t:value-of select="@attr"/>
  		        <t:text> * </t:text>
  	      </t:for-each>
      </out>
   </t:template>
</t:transform>
