<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.baseuri.exmpl/tests/"
             version="2.0">
<!-- Purpose: Test with fn:base-uri($arg) on a temporary tree that consists 
  				entirely of locally constructed elements and attributes.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="temptree" xml:base="d://tests/">
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <b>Text1</b>
         <c>helloText2</c>
      </a>
   </t:variable>

   <t:template match="/">
      <out>
         <t:value-of select="base-uri($temptree)"/>
      </out>
   </t:template>
</t:transform>
