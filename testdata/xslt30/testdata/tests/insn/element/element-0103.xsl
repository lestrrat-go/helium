<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that duplicate namespace nodes are removed. -->

   <t:template match="doc">
		    <out>
         <t:element xmlns:abc="http://www.test.com" name="name">
   			      <t:namespace name="abc">http://www.test.com</t:namespace>
		       </t:element>
      </out>
	  </t:template>
</t:transform>
