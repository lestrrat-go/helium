<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
    xmlns:s="http://www.w3.org/XQueryTest/unionListDefined" exclude-result-prefixes="#all">

    <!-- castable operator with list types. Tests created by manual conversion of QT3 tests Castable-ListType-* -->

    <xsl:import-schema namespace="http://www.w3.org/XQueryTest/unionListDefined"
        schema-location="unionListDefined.xsd"/>

    <xsl:function name="s:lazy">
        <xsl:param name="p"/>
        <xsl:sequence
            select="
                if (current-date() gt xs:date('1900-01-01')) then
                    $p
                else
                    '*!*!*!*'"
        />
    </xsl:function>

    <xsl:variable name="qt3tests">
        <test-case name="Castable-ListType-2" covers-30="cast-to-list">
            <description>Cast the expression to the defined list type. The test should fail as the
                base type for the list is decimal</description>
            <created by="O'Neil Delpratt" on="2011-10-27"/>
            <modified by="Michael Kay" on="2015-01-06" change="drop version declaration from query"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "1.1 2.1 3.1" castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("1.1 2.1 3.1") castable as s:intListType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-3" covers-30="cast-to-list">
            <description>Test castability of xs:string to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "1 2 3" castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("1 2 3") castable as s:intListType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-4" covers-30="cast-to-list">
            <description>Test castability of xs:string to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "" castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy("") castable as s:intListType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-5" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("1.1 2.1 3.1") castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("1.1 2.1 3.1")) castable as s:intListType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-6" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("1 2 3") castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("1 2 3")) castable as s:intListType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-7" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("") castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("")) castable as s:intListType1'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-8" covers-30="cast-to-list">
            <description>Test castability of xs:integer sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        (1, 2, 3) castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy((1, 2, 3)) castable as s:intListType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-9" covers-30="cast-to-list">
            <description>Test castability of empty sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <modified by="Tim Mills" on="2016-04-25" change="see bugs 29571, 29584"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        () castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(()) castable as s:intListType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-10" covers-30="cast-to-list">
            <description>Test castability of s:intListType1 sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        s:intListType1("1 2 3") castable as s:intListType1]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:intListType1("1 2 3")) castable as s:intListType1'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-11" covers-30="cast-to-list">
            <description>Test castability of xs:string to list type. The test should fail as the
                base type for the list is decimal</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "1.1 2.1 3.1" castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("1.1 2.1 3.1") castable as s:intListType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-12" covers-30="cast-to-list">
            <description>Test castability of xs:string to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "1 2 3" castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("1 2 3") castable as s:intListType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-13" covers-30="cast-to-list">
            <description>Test castability of xs:string to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        "" castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy("") castable as s:intListType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-14" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("1.1 2.1 3.1") castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("1.1 2.1 3.1")) castable as s:intListType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-15" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("1 2 3") castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("1 2 3")) castable as s:intListType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-16" covers-30="cast-to-list">
            <description>Test castability of xs:untypedAtomic to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        xs:untypedAtomic("") castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(xs:untypedAtomic("")) castable as s:intListType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-17" covers-30="cast-to-list">
            <description>Test castability of xs:integer sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        (1, 2, 3) castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy((1, 2, 3)) castable as s:intListType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-18" covers-30="cast-to-list">
            <description>Test castability of empty sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        () castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(()) castable as s:intListType1?'/></test2>
            <result>
                <assert-true/>
            </result>
        </test-case>

        <test-case name="Castable-ListType-19" covers-30="cast-to-list">
            <description>Test castability of s:intListType1 sequence to list type.</description>
            <created by="Tim Mills" on="2016-04-11"/>
            <environment ref="union-List-defined-Types"/>
            <dependency type="spec" value="XQ30+"/>
            <dependency type="feature" value="schemaImport"/>
            <test><![CDATA[
       import schema namespace s="http://www.w3.org/XQueryTest/unionListDefined";
        s:intListType1("1 2 3") castable as s:intListType1?]]>
            </test>
            <test2><xsl:value-of select='s:lazy(s:intListType1("1 2 3")) castable as s:intListType1?'/></test2>
            <result>
                <assert-false/>
            </result>
        </test-case>
    </xsl:variable>

    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:for-each
                select="$qt3tests/test-case[(result/assert-true and not(test2 = 'true')) or (result/assert-false and not(test2 = 'false'))]">
                <failure test="{@name}"/>
            </xsl:for-each>
        </out>
    </xsl:template>

</xsl:stylesheet>
