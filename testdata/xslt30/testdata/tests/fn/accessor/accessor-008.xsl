<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for the equality "fn:doc(fn:document-uri($arg)) is $arg", where $arg is a document node -->

   <t:template match="/">
      <t:variable name="temp1" select="."/>
      <t:variable name="temp2">
	        <doc>top<t:processing-instruction name="pi">PI_data</t:processing-instruction>
            <t:comment>This is the 1st comment</t:comment>
            <t:text>text-in-doc</t:text>
            <inner/>
         </doc>
      </t:variable>
      <out>
         <t1>
            <t:value-of select="doc(document-uri($temp1)) is $temp1"/>
         </t1>
      </out>
   </t:template>
</t:transform>
