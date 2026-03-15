<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
      <!-- Purpose:  Tests using function-available for generate-id,system-property, key, current
  				unparsed-entity-uri, and unparsed-entity-public-id in a use-when instruction. -->

      <t:template match="/">
            <out>
                  <t:call-template name="temp"/>
                  <in t:use-when="function-available('current')" f="current">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
                  <in t:use-when="function-available('generate-id')" f="generate-id">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
                  <in t:use-when="function-available('key')" f="key">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
                  <in t:use-when="function-available('unparsed-entity-uri')" f="unparsed-entity-uri">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
                  <in t:use-when="function-available('unparsed-entity-public-id')"
                        f="unparsed-entity-public-id">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
                  <in t:use-when="function-available('system-property')" f="system-property">
                        <t:value-of select="'use-when evaluates to true!!!'" />
                  </in>
            </out>
      </t:template>

      <t:template name="temp">
            <fa f="current">
                  <t:value-of select="function-available('current')"/>
            </fa>
            <fa f="generate-id">
                  <t:value-of select="function-available('generate-id')"/>
            </fa>
            <fa f="key">
                  <t:value-of select="function-available('key')"/>
            </fa>
            <fa f="unparsed-entity-uri">
                  <t:value-of select="function-available('unparsed-entity-uri')"/>
            </fa>
            <fa f="unparsed-entity-public-id">
                  <t:value-of select="function-available('unparsed-entity-public-id')"/>
            </fa>
            <fa f="system-property">
                  <t:value-of select="function-available('system-property')"/>
            </fa>
      </t:template>
</t:transform>
