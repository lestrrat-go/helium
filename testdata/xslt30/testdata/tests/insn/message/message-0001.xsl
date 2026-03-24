<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test output of a message that cannot be serialized.
       According to XSLT 2.0 second edition (following erratum E20) this
       is a recoverable error, so this test case may either generate an error
       or output the element <out/>. XSLT 3.0 says that the transformation does not fail,
       though the output is left unspecified -->

   <t:variable name="v">
        <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform" att="An attribute">boo</a>
    </t:variable>

   <t:template name="main">
   		<out>
         <t:message select="$v//@att"/>
      </out>
	  </t:template>
</t:transform>
