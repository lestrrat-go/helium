<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in @collation of xsl:for-each-group.
				Cases for AVT: no literal element, LE at start and end 
				AVT value comes from a node or a variable.-->

   <t:variable name="var1" select="/doc/str1"/>

   <t:variable name="var2" select="/doc/str4"/>

   <t:template match="/doc">
      <out>
         <t:text>
</t:text>
         <t:for-each-group select="item" group-by="@attr" collation="{str1}">
		          <t:value-of select="."/>
	        </t:for-each-group>
         <t:text>
</t:text>
         <t:for-each-group select="item" group-by="@attr" collation="{$var1}">
		          <t:value-of select="."/>
	        </t:for-each-group>
         <t:text>
</t:text>
         <t:for-each-group select="item"
                           group-by="@attr"
                           collation="http://{str4}collation/codepoint">
		          <t:value-of select="."/>
	        </t:for-each-group>
         <t:text>
</t:text>
         <t:for-each-group select="item"
                           group-by="@attr"
                           collation="http://{$var2}collation/codepoint">
		          <t:value-of select="."/>
	        </t:for-each-group>
      </out>
   </t:template>
</t:transform>
