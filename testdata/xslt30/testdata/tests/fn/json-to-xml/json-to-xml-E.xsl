<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <xsl:template name="json-to-xml-error-001">
        <xsl:variable name="metadata">
            <description>Empty string</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json" select="''"/>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-002">
        <xsl:variable name="metadata">
            <description>Unknown token</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{error}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-003">
        <xsl:variable name="metadata">
            <description>Unknown token</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{terriblyWrong}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-004">
        <xsl:variable name="metadata">
            <description>Extra token</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{}"extra token"</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-005">
        <xsl:variable name="metadata">
            <description>Non-string key</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{1:1}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-006">
        <xsl:variable name="metadata">
            <description>Non-string key</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{true:1}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-007">
        <xsl:variable name="metadata">
            <description>Missing colon after key</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{"key" 1}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-008">
        <xsl:variable name="metadata">
            <description>Extra token after map value</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{"key":1 2}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-009">
        <xsl:variable name="metadata">
            <description>Extra token after array value</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["key",1 2]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-010">
        <xsl:variable name="metadata">
            <description>Leading + on number</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">[+1]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-011">
        <xsl:variable name="metadata">
            <description>Redundant leading 0 on number</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">[01]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-012">
        <xsl:variable name="metadata">
            <description>Hex number</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">[1234A]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-013">
        <xsl:variable name="metadata">
            <description>Trailing backslash</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["123\"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-014">
        <xsl:variable name="metadata">
            <description>Short unicode</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["123\u234"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-015">
        <xsl:variable name="metadata">
            <description>Unicode null</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String with \u0000 backspace"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-016">
        <xsl:variable name="metadata">
            <description>Trailing comma in object</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{"key":123,}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-017">
        <xsl:variable name="metadata">
            <description>Trailing comma in object - liberal spec</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="John Lumley" on="2014-07-22"
                change="option 'spec':'liberal' altered to 'liberal':true()"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">{"key":123 ,}</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'liberal':true()})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-018">
        <xsl:variable name="metadata">
            <description>Trailing comma in array</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["key",123,]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json)"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-019">
        <xsl:variable name="metadata">
            <description>Trailing comma in array - liberal spec</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="John Lumley" on="2014-07-22"
                change="option 'spec':'liberal' altered to 'liberal':true()"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["key",123,]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'liberal':true()})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-020">
        <xsl:variable name="metadata">
            <description>Liberal option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="John Lumley" on="2014-07-22" change="option 'spec' altered to 'liberal'"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'liberal':()})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-021">
        <xsl:variable name="metadata">
            <description>Liberal option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="John Lumley" on="2014-07-22" change="option 'spec' altered to 'liberal'"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'liberal':'some string'})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-022">
        <xsl:variable name="metadata">
            <description>Validate option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json, map{'validate':()})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-023">
        <xsl:variable name="metadata">
            <description>Validate option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json, map{'validate':(true(),true())})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-024">
        <xsl:variable name="metadata">
            <description>Validate option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json, map{'validate':'EMCA-262'})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-025">
        <xsl:variable name="metadata">
            <description>Unescape option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="Debbie Lockett" on="2015-08-12" change="Change keyword unescape->escape"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json, map{'escape':()})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-026">
        <xsl:variable name="metadata">
            <description>Escape option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="Debbie Lockett" on="2015-08-12" change="Change keyword unescape->escape"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'escape':(true(),true())})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-027">
        <xsl:variable name="metadata">
            <description>Escape option error</description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="Debbie Lockett" on="2015-08-12" change="Change keyword unescape->escape"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json, map{'escape':'EMCA-262'})"/>
    </xsl:template>
    <xsl:template name="json-to-xml-error-028">
        <xsl:variable name="metadata">
            <description>Fallback option not function </description>
            <created by="John Lumley" on="2013-10-01"/>
            <modified by="Abel Braaksma" on="2014-10-21" change="':=' for maps is replaced by ':'" />
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:variable name="json">["String"]</xsl:variable>
        <xsl:copy-of select="json-to-xml($json,map{'fallback':'dummy'})"/>
    </xsl:template>
</xsl:stylesheet>
