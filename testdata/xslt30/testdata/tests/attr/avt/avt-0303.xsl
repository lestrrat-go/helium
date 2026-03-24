<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test AVT in the name attribute of xsl:namespace. AVT value comes from a node or a variable.
  				Cases for AVT:
				-no literal element
				-literal element at start
				-literal element at end
				-literal element at start and end
				-literal element present and value obtained at run time is empty sequence-->

   <t:variable name="var" select="/doc/str[2]"/>

   <t:template match="/doc">
      <out>
         <t:element name="elem">
            <t:namespace name="{str[1]}" select="'http://www.test.org/'"/>
            <t:namespace name="{$var}" select="'http://www.example.org/'"/>
         </t:element>
         <t:element name="elem">
            <t:namespace name="{str[1]}bc" select="'http://www.test.org/'"/>
            <t:namespace name="{$var}cd" select="'http://www.example.org/'"/>
         </t:element>
         <t:element name="elem">
            <t:namespace name="a{str[1]}" select="'http://www.test.org/'"/>
            <t:namespace name="a{$var}c" select="'http://www.example.org/'"/>
         </t:element>
         <t:element name="elem">
            <t:namespace name="a{str[4]}" select="'http://www.test.org/'"/>
         </t:element>
      </out>
   </t:template>
</t:transform>
