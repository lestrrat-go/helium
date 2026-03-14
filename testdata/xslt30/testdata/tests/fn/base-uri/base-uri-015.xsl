<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Testcase with fn:base-uri($arg) where $arg is an attribute, 
  				text or comment node created from scratch and with no parents.-->

   <t:variable name="attrib" as="attribute()">
		    <t:attribute name="a" select="'attribute'"/>
	  </t:variable>

   <t:variable name="text" as="text()">
		    <t:text>simple text</t:text>
	  </t:variable>

   <t:variable name="comment" as="comment()">
		    <t:comment select="'This is a comment'"/>
	  </t:variable>

   <t:template match="/" xml:base="/main/">
		    <out>
         <example1>
            <t:value-of select="base-uri($attrib)"/>
         </example1>
         <example2>
            <t:value-of select="base-uri($text)"/>
         </example2>
         <example3>
            <t:value-of select="base-uri($comment)"/>
         </example3>
      </out>
	  </t:template>
</t:transform>
