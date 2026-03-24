<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:j="http://www.w3.org/2005/xpath-functions">
    <xsl:param name="options.escape" select="map{'escape':true()}"/>
    <xsl:param name="options.validate" select="map{'validate':true()}"/>

    <!--<xsl:param name="options.rfc" select="map{'spec':'RFC4627'}"/>
    <xsl:param name="options.ecma" select="map{'spec':'ECMA-262'}"/>-->

    <xsl:param name="s0.map" select="'{}'"/>
    <xsl:param name="s0.array" select="'[]'"/>
    <xsl:param name="s1.number" select="'[1]'"/>
    <xsl:param name="s1.string" select="'[&#34;a&#34;]'"/>
    <xsl:param name="s1.true" select="'[true]'"/>
    <xsl:param name="s1.false" select="'[false]'"/>
    <xsl:param name="s1.null" select="'[null]'"/>

    <xsl:param name="s2" select="'{&#34;x&#34;:1, &#34;y&#34;:[1,2,3]}'"/>

    <xsl:param name="escape.1" select="'[&#34;Data with \&#34; within it&#34;]'"/>
    <xsl:param name="escape.2"
        select="'{&#34;Key \&#34; with quote&#34;:&#34;Data with \&#34; within it&#34;}'"/>
    <xsl:param name="escape.3" select="'[&#34;Data with \u000C within it&#34;]'"/>
    <xsl:param name="escape.4"
        select="'{&#34;Key \u000C with special character&#34;:&#34;Data with \f within it&#34;}'"/>


    <xsl:template name="json-to-xml-001">
        <xsl:variable name="metadata">
            <description>Empty object as document-node</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s0.map) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>document-node()</assert-type>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-002">
        <xsl:variable name="metadata">
            <description>Empty object</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s0.map) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}map)</assert-type>
                <assert>count($result/*) eq 0</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-003">
        <xsl:variable name="metadata">
            <description>Empty array as document-node</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s0.array) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>document-node()</assert-type>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-004">
        <xsl:variable name="metadata">
            <description>Empty array</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s0.array) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}array)</assert-type>
                <assert>count($result/*) eq 0</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-005">
        <xsl:variable name="metadata">
            <description>Array Number</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s1.number) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}array)</assert-type>
                <assert>count($result/*) eq 1</assert>
                <assert>$result/*[1] instance of element(j:number,xs:untyped)</assert>
                <assert>xs:double($result/*[1]) eq 1</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-006">
        <xsl:variable name="metadata">
            <description>Array String</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s1.string) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(*)</assert-type>
                <assert>count($result/*) eq 1</assert>
                <assert>$result/*[1] instance of element(j:string,xs:untyped)</assert>
                <assert>xs:string($result/*[1]) eq "a"</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-007">
        <xsl:variable name="metadata">
            <description>Array True</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s1.true) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(*)</assert-type>
                <assert>count($result/*) eq 1</assert>
                <assert>$result/*[1] instance of element(j:boolean,xs:untyped)</assert>
                <assert>xs:boolean($result/*[1]) eq true()</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-008">
        <xsl:variable name="metadata">
            <description>Array False</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s1.false) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(*)</assert-type>
                <assert>count($result/*) eq 1</assert>
                <assert>$result/*[1] instance of element(j:boolean,xs:untyped)</assert>
                <assert>xs:boolean($result/*[1]) eq false()</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-009">
        <xsl:variable name="metadata">
            <description>Array Null</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($s1.null) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(*)</assert-type>
                <assert>count($result/*) eq 1</assert>
                <assert>$result/*[1] instance of element(j:null,xs:untyped)</assert>
                <assert>empty($result/*[1]/(*|text()))</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-010">
        <xsl:variable name="metadata">
            <description>Larger 2</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml(unparsed-text('data001.json')) "/>
        <!--<all-of>
                <assert-count>1</assert-count>
                <assert-type>element(*)</assert-type>
                <assert>count($result/*) eq 1</assert>
            </all-of>-->
    </xsl:template>
    <xsl:template name="json-to-xml-011">
        <xsl:variable name="metadata">
            <description>Larger 2</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml(unparsed-text('data005.json')) "/>
    </xsl:template>
    <xsl:template name="json-to-xml-012">
        <xsl:variable name="metadata">
            <description>Larger 2</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml(unparsed-text('escapeText.json')) "/>
    </xsl:template>

    <!-- Escaping tests -->
    <xsl:template name="json-to-xml-escape-001">
        <xsl:variable name="metadata">
            <description>Escaped quote string</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.1)//j:string "/>
        <!--<test> json-to-xml($escape.1)//j:string</test>
            <result>
                <all-of>
                    <!-\-<assert-count>1</assert-count>
                    <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}string)</assert-type>
                    <assert-string-value>Data with " within it</assert-string-value>
                    <any-of>
                        <assert>empty($result/@escaped)</assert>
                        <assert>$result/@escaped eq "false"</assert>
                    </any-of>
                    <any-of>
                        <assert>empty($result/@escaped-key)</assert>
                        <assert>$result/@escaped-key eq "false"</assert>
                    </any-of>-\->
                </all-of>
            </result>-->
    </xsl:template>
    <xsl:template name="json-to-xml-escape-002">
        <xsl:variable name="metadata">
            <description>Escaped quote and key string</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.2)//j:string "/>
        <!--<test> json-to-xml($escape.2)//j:string</test>
            <result>
                <all-of>
                    <!-\-<assert-count>1</assert-count>
                    <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}string)</assert-type>
                    <assert-string-value>Data with " within it</assert-string-value>
                    <assert>$result/@key eq 'Key " with quote'</assert>
                    <any-of>
                        <assert>empty($result/@escaped)</assert>
                        <assert>$result/@escaped eq "false"</assert>
                    </any-of>
                    <any-of>
                        <assert>empty($result/@escaped-key)</assert>
                        <assert>$result/@escaped-key eq "false"</assert>
                    </any-of>-\->
                </all-of>
            </result>-->
    </xsl:template>
    <xsl:template name="json-to-xml-escape-003">
        <xsl:variable name="metadata">
            <description>Escaped quote string - remaining escaped</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.1,$options.escape)//j:string "/>
        <!--<test> json-to-xml($escape.1,$options.escape)//j:string</test>
            <result>
                <!-\- <all-of>
                    <assert-count>1</assert-count>
                    <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}string)</assert-type>
                    <assert-string-value>Data with \" within it</assert-string-value>
                    <assert>$result/@escaped eq "true"</assert>
                    <any-of>
                        <assert>empty($result/@escaped-key)</assert>
                        <assert>$result/@escaped-key eq "false"</assert>
                    </any-of>
                </all-of>-\->
            </result>-->
    </xsl:template>
    <xsl:template name="json-to-xml-escape-004">
        <xsl:variable name="metadata">
            <description>Escaped quote string and key - remaining escaped</description>
            <created by="John Lumley" on="2013-10-01"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.2,$options.escape)//j:string "/>
        <!--<test> json-to-xml($escape.2,$options.escape)//j:string</test>
            <result>
                <!-\-<all-of>
                    <assert-count>1</assert-count>
                    <assert-type>element(Q{http://www.w3.org/2005/xpath-functions}string)</assert-type>
                    <assert-string-value>Data with \" within it</assert-string-value>
                    <assert>$result/@key eq 'Key \" with quote'</assert>
                    <assert>$result/@escaped eq "true"</assert>
                    <assert>$result/@escaped-key eq "true"</assert>
                </all-of>-\->
            </result>-->
    </xsl:template>
    <xsl:template name="json-to-xml-escape-005">
        <xsl:variable name="metadata">
            <description>Escaped character string - remaining escaped</description>
            <created by="Debbie Lockett" on="2015-08-12"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.3,$options.escape)//j:string "/>
        <!--<test> json-to-xml($escape.3,$options.escape)//j:string</test>
            <result>
                <!-\- <all-of>
                <assert>* instance of element(Q{http://www.w3.org/2005/xpath-functions}string)</assert>
                <assert>. = 'Data with \f within it'</assert>
                <assert>*/@escaped eq "true"</assert>
                <any-of>
                    <assert>empty(*/@escaped-key)</assert>
                    <assert>*/@escaped-key eq "false"</assert>
                </any-of>
            </all-of>-\->
            </result>-->
    </xsl:template>
    <xsl:template name="json-to-xml-escape-006">
        <xsl:variable name="metadata">
            <description>Escaped character string and key - remaining escaped</description>
            <created by="Debbie Lockett" on="2015-08-12"/>
            <environment ref="JSON-XML"/>
        </xsl:variable>
        <xsl:copy-of select=" json-to-xml($escape.4,$options.escape)//j:string "/>
        <!--<test> json-to-xml($escape.4,$options.escape)//j:string</test>
            <result>
                <!-\-<all-of>
                <assert>* instance of element(Q{http://www.w3.org/2005/xpath-functions}string)</assert>
                <assert>. = 'Data with \f within it'</assert>
                <assert>*/@key eq 'Key \f with special character'</assert>
                <assert>*/@escaped eq "true"</assert>
                <assert>*/@escaped-key eq "true"</assert>
            </all-of>-\->
            </result>-->
    </xsl:template>
</xsl:stylesheet>
