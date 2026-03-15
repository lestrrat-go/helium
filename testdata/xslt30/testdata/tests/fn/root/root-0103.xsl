<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Function is invoked with non-existant element node -->

   <t:template match="/">
	 	   <out>
         <t:value-of select="root( testnode )"/>
         <t:value-of select="root( () )"/>
         <t:value-of select="root( doc/string1/testingtest )"/>
      </out>
	  </t:template>
</t:transform>
