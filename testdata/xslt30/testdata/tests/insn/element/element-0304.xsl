<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that adjacent strings/text nodes copied into content result 
                in one merged text node. -->

   <t:variable name="x" as="element()*">
  	   <node xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="()"/>
         <t:text/>
         <t:value-of select="''"/>
         <t:text>x_9</t:text>
         <t:text>The</t:text>
         <t:text>last </t:text>
         <t:text>one</t:text>
         <t:value-of select="()"/>
         <x_6>
            <t:value-of select="''"/>
         </x_6>
         <a>text</a>
      </node>
  </t:variable>

   <t:template match="doc">
		    <out>
         <t:apply-templates select="$x/node()"/>
      </out>
	  </t:template>

   <t:template match="text()">
		    <t:text>
Number of text node: </t:text>
      <t:value-of select="count($x/text())"/>
	  </t:template>

   <t:template match="element()">
		    <t:text>
Number of element node: </t:text>
      <t:value-of select="count($x/element())"/>
	  </t:template>
</t:transform>
