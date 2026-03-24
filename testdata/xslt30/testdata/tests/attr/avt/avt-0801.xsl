<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test AVT in the ordinal attribute of xsl:number. AVT value comes from a  variable.-->

   <t:variable name="lang" select="/doc/item[1]"/>

   <t:variable name="lv" select="/doc/item[2]"/>

   <t:variable name="ord" select="/doc/item[3]"/>

   <t:template match="/doc">
      <out>
         <t:for-each select="num">
  		        <t:number value="."
                      lang="{$lang}"
                      format="Ww"
                      letter-value="{$lv}"
                      ordinal="{$ord}"/>
  	         <t:text> * </t:text>
  	      </t:for-each>
      </out>
   </t:template>
</t:transform>
