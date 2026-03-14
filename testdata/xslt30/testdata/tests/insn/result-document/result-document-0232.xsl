<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://example.example.org/ns/one"
             xmlns:two="http://example.example.org/ns/two"
             xmlns:three="http://example.example.org/ns/three"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test with xsl:result-document and @undeclare-prefixes="yes", @method="text". 
  				Show that attribute is ignored.-->

   <t:variable name="temp">
	     <one:a>a1</one:a>
	     <two:b>b2</two:b>
   </t:variable>

   <t:template match="/">
  	   <t:result-document method="text" undeclare-prefixes="yes">
      	  <out>
            <t:value-of select="$temp/one:a"/>
            <t:value-of select="$temp/two:b"/>
         </out>
      </t:result-document>
   </t:template>
</t:transform>
