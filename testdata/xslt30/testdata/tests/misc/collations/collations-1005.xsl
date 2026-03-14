<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: To verify that the first recognized collation URI value from a list specified in a default-collation attribute is returned by fn:default-collation.-->

   <t:template match="doc"
               default-collation="http://www.none1.org http://www.none2.org http://www.w3.org/2005/xpath-functions/collation/codepoint http://frenchcollation.org">
      <out>
         <t:value-of select="default-collation()"/>
      </out> 
   </t:template>
</t:transform>
