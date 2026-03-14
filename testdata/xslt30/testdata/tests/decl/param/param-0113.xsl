<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
      <!-- Purpose: Test case that uses <xsl:param> instruction with a required="yes" attribute, 
                while the instruction has a non-empty sequence constructor. -->

      <t:template match="doc">
            <t:param name="par1" required="yes">test param</t:param>
            <out> Parameter Value: <t:value-of select="$par1"/></out>
      </t:template>
</t:transform>
