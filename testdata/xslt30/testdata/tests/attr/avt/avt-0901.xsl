<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in @lang, @order and @collation of xsl:sort. AVT value 
  				has no literal element and comes from a node or a variable.-->

   <t:variable name="lang" select="/doc/data[1]"/>

   <t:variable name="collation" select="/doc/data[3]"/>

   <t:template match="/doc">
      <out>
         <t:for-each select="item">
  		        <t:sort select="@target"
                    lang="{$lang}"
                    collation="{$collation}"
                    order="{data[2]}"/>
  		        <t:value-of select="@num"/>
  	         <t:text> * </t:text>
  	      </t:for-each>
      </out>
   </t:template>
</t:transform>
