<?xml version="1.1" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses character references #x1 through #x1F in the text nodes 
  				inside an LRE in the stylesheet. These characteres were not allowed in the xml 1.0.
  				No error should be raised when the xml declaration has version=1.1 and the characters 
  				should be serialized to the output document.  -->

   <t:output method="xml" version="1.1"/>

   <t:template match="/">
	     <out>
         <t:text>&#x1;</t:text>
         <t:text>&#x2;</t:text>
         <t:text>&#x3;</t:text>
         <t:text>&#x4;</t:text>
         <t:text>&#x5;</t:text>
         <t:text>&#x6;</t:text>
         <t:text>&#x7;</t:text>
         <t:text>&#x8;</t:text>
         <t:text>	</t:text>
         <t:text>
</t:text>
         <t:text>&#xb;</t:text>
         <t:text>&#xc;</t:text>
         <t:text>&#xD;</t:text>
         <t:text>&#xe;</t:text>
         <t:text>&#xf;</t:text>
         <t:text>&#x10;</t:text>
         <t:text>&#x11;</t:text>
         <t:text>&#x12;</t:text>
         <t:text>&#x13;</t:text>
         <t:text>&#x14;</t:text>
         <t:text>&#x15;</t:text>
         <t:text>&#x16;</t:text>
         <t:text>&#x17;</t:text>
         <t:text>&#x18;</t:text>
         <t:text>&#x19;</t:text>
         <t:text>&#x1a;</t:text>
         <t:text>&#x1b;</t:text>
         <t:text>&#x1c;</t:text>
         <t:text>&#x1d;</t:text>
         <t:text>&#x1e;</t:text>
         <t:text>&#x1f;</t:text>
      </out>
  </t:template>
</t:transform>
