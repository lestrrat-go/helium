<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that Zero-length text nodes within the xsl:comment are removed. -->

   <t:template match="doc">
		    <t:text>
</t:text>
		    <t:variable name="zero-length-t1" as="text()">
  			    <t:value-of select="''"/>
		    </t:variable>
		    <t:variable name="zero-length-t2" as="text()">
  			    <t:text/>
		    </t:variable>
   		<out>
         <t:text>
</t:text>
         <t:comment>
   			      <t:value-of separator="*"
                        select="('Old', '', 'est', $zero-length-t1, 'att', '', 'rib', $zero-length-t2,'ute')"/>
   		    </t:comment>
         <t:text>
</t:text>
      </out>
	  </t:template>
</t:transform>
