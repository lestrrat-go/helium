<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in a LRE where the value comes from a temporary tree. -->

   <t:variable name="var">
	     <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <item>att1</item>
         <item>att2</item>
         <item>att3</item>
         <item>att4</item>
      </doc>
   </t:variable>

   <t:template match="doc">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <out test="{$var}"/>
         <out test="{$var/doc/item}"/>
      </output>
   </t:template>
</t:transform>
