<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in the separator attribute of xsl:attribute 
  				where the value comes from a set of nodes. 
  				(literal elements are present in AVT at start and end)-->

   <t:template match="/">
      <out>
         <t:element name="name">
		          <t:attribute name="abc" separator=";{doc/item};" select="doc/str"/>
	        </t:element>
      </out>
   </t:template>
</t:transform>
