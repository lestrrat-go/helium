<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test case that uses the NEL (#x85) character in an stylesheet.
  				Use version="1.1" on the xml declaration of the stylesheet, on the input 
  				document and on the xsl:output instruction.  -->

   <t:output method="xml" version="1.1"/>

   <t:template match="/">
	     <out>
		&#x85;
		<t:element name="a1">This is &#x85; gonna be fun!</t:element>
		&#x85;
		<a2>
            <t:value-of select="doc/a"/>
         </a2>
		&#x85;
	</out>
  </t:template>
</t:transform>
