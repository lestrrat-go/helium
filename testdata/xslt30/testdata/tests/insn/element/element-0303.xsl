<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that zero-length strings result in no text node child. -->

   <t:variable name="x" as="element()">
  	   <node xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:text>x_1</t:text>
         <x_2>
            <t:text/>
         </x_2>
         <t:text/>
         <x_4>
            <t:value-of select="()"/>
         </x_4>
         <t:value-of select="()"/>
         <x_6>
            <t:value-of select="''"/>
         </x_6>
         <t:value-of select="''"/>
         <x_8/>
         <t:text>x_9</t:text>
      </node>
  </t:variable>

   <t:template match="doc">
		    <out>
         <t:apply-templates select="$x/node()"/>
      </out>
	  </t:template>

   <t:template match="text()">
		    <t:text>
found text node: </t:text>
      <t:value-of select="."/>
	  </t:template>

   <t:template match="element()">
		    <t:text>
found element node: </t:text>
      <t:copy-of select="."/>
		    <t:apply-templates/>
	  </t:template>
</t:transform>
